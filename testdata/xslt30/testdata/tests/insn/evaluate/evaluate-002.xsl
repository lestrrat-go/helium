<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:my="http://www.example.com/my-functions" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs my">


    <!-- formerly saxon007 -->

    <!-- rewritten Nov 2009 to use xsl:evaluate -->

    <!--
This stylesheets illustrates a problem in saxon 7.8.

Its occurs using my:evaluate and unions.
                 ==============     ======
There are two manifestions: sometimes the program crashes, and sometimes results are wrong.
-->

    <!-- get the settings from the two files -->
    <xsl:variable name="settings1">
        <settings>
            <setting1 v="value1"/>
        </settings>
    </xsl:variable>

    <xsl:variable name="settings2">
        <settings>
            <setting2 v="value2"/>
        </settings>
    </xsl:variable>

    <!-- combine all settings into one node set, this could be done in several ways
     here are two ways -->

    <xsl:variable name="settings" select="($settings1 | $settings2)/settings"/>
    <xsl:variable name="settingsalt" select="($settings1/settings) | ($settings2/settings)"/>

    <!-- for comparison, take the same settings from a single file, so that union
    is not used -->

    <xsl:variable name="settingscombined">
        <settings>
            <setting1 v="value1"/>
            <setting2 v="value2"/>
        </settings>
    </xsl:variable>
    <xsl:variable name="settingsall" select="$settingscombined/settings"/>

    <!-- it will turn out that it works better if we descend to the settings level later on
    when we use the node sets -->
    <xsl:variable name="settingstop" select="$settings1 | $settings2"/>
    <xsl:variable name="settingstoprev" select="$settings2 | $settings1"/>

    <!-- the problem does not occur when we fetch the settings statically, but only
    if we use my:evaluate -->
    <xsl:variable name="name1" select="'setting1'"/>
    <xsl:variable name="name2" select="'setting2'"/>

    <!-- we need to get an ' inside a my:expression -->
    <xsl:variable name="apos">'</xsl:variable>

    <xsl:template name="main">
        <out>

            <!--static fetching of values-->

            <z>STATIC setting1=[<xsl:value-of select="$settings/setting1/@v"/>]</z>
            <z>STATIC setting2=[<xsl:value-of select="$settings/setting2/@v"/>]</z>

            <!-- yields expected results:

STATIC setting1=[value1]
STATIC setting2=[value2]

    -->

            <!--dynamic fetching of values from $settings-->


            <a>DYNAMIC(A-all) setting1=[<xsl:value-of
                    select="my:evaluate(concat('$p1/',$name1,'/@v'),$settingsall)"/>]</a>
            <a>DYNAMIC(A-all) setting2=[<xsl:value-of
                    select="my:evaluate(concat('$p1/',$name2,'/@v'),$settingsall)"/>]</a>

            <!-- output is normal:

DYNAMIC(A-all) setting1=[value1]
DYNAMIC(A-all) setting2=[value2]

    -->

            <!--dynamic fetching of values from $settingstop: we descend to the settings level in the second
        parameter of my:evaluate -->

            <a>DYNAMIC(B) count=<xsl:value-of select="count($settingstop/settings)"/></a>
            <xsl:for-each select="$settingstop/settings/*">
                <b>DYNAMIC(B-for) <xsl:value-of select="concat(name(),'=[',@v,']')"/></b>
            </xsl:for-each>
            <d>DYNAMIC(B-top) setting1=[<xsl:value-of
                    select="my:evaluate(concat('$p1/',$name1,'/@v'),$settingstop/settings)"/>]</d>
            <d>DYNAMIC(B-top) setting2=[<xsl:value-of
                    select="my:evaluate(concat('$p1/',$name2,'/@v'),$settingstop/settings)"/>]</d>
            <d>DYNAMIC(B-1) setting1=[<xsl:value-of
                    select="my:evaluate(concat('$p1/',$name1,'/@v'),$settings1/settings)"/>]</d>
            <d>DYNAMIC(B-2) setting2=[<xsl:value-of
                    select="my:evaluate(concat('$p1/',$name2,'/@v'),$settings2/settings)"/>]</d>


            <!--dynamic fetching of values from $settingstoprev-->

            <e>DYNAMIC(C) count=<xsl:value-of select="count($settingstoprev/settings)"/></e>
            <xsl:for-each select="$settingstoprev/settings/*">
                <f>DYNAMIC(C-for) <xsl:value-of select="concat(name(),'=[',@v,']')"/></f>
            </xsl:for-each>
            <g>DYNAMIC(C-top) setting1=[<xsl:value-of
                    select="my:evaluate(concat('$p1/',$name1,'/@v'),$settingstoprev/settings)"
                />]</g>
            <g>DYNAMIC(C-top) setting1=[<xsl:value-of
                    select="my:evaluate(concat('$p1/',$name2,'/@v'),$settingstoprev/settings)"
                />]</g>


            <!--dynamic fetching of values from other ways to specify the nodes-->

            <h>DYNAMIC(D) setting1=[<xsl:value-of
                    select="my:evaluate(concat('$p1[name()=',$apos,$name1,$apos,']/@v'),($settings1 | $settings2)/settings/*)"
                />]</h>
            <h>DYNAMIC(D) setting1=[<xsl:value-of
                    select="my:evaluate(concat('$p1[name()=',$apos,$name2,$apos,']/@v'),($settings1 | $settings2)/settings/*)"
                />]</h>

        </out>
    </xsl:template>

    <xsl:function name="my:evaluate" as="xs:string">
        <xsl:param name="expr" as="xs:string"/>
        <xsl:param name="param" as="item()*"/>
        <xsl:evaluate xpath="$expr" as="xs:string">
            <xsl:with-param name="p1" select="$param"/>
        </xsl:evaluate>
    </xsl:function>

</xsl:stylesheet>
