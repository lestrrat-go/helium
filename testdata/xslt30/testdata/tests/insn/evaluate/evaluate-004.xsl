<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0" 
        xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
        xmlns:xs="http://www.w3.org/2001/XMLSchema" 
        xmlns:xfn="http://www.w3.org/2005/xpath-functions"
        xmlns:h="http://www.xyz.com/h"
        exclude-result-prefixes="h xsl xs  xfn"
>

<!-- A more complex example of xsl:evaluate with multiple parameters --> 

<xsl:variable name="resource">
    <res>
        <resgrp>
                <thing id="a" color="red" size="big" weight="heavy"/>
                <thing id="b" color="green" size="any" weight="any"/>
                <thing id="c" color="blue" size="small" weight="light"/>
                <thing id="d" color="red" size="small" weight="heavy"/>
                <thing id="e" color="green" size="small" weight="heavy"/>
                <thing id="f" color="blue" size="big" weight="heavy"/>
                <thing id="g" color="red" size="big" weight="light"/>
                <thing id="h" color="green" size="small" weight="light"/>
                <thing id="i" color="blue" size="small" weight="heavy"/>
        </resgrp>
    </res>
</xsl:variable>

<xsl:template name="main">
    <xsl:variable name="queryseq">
        <aa>red big heavy</aa>
        <aa>blue small light</aa>
        <aa>red small light</aa>
        <aa>green big light</aa>
    </xsl:variable>
    <out>
    <xsl:for-each select="$queryseq/*">
        <xsl:variable name="parms" select="."/>
        <xsl:copy-of select="h:getresource('//thing[@color = $p1 and @size = $p2 and @weight = $p3]', '@id', $parms)//@id"/>
    </xsl:for-each>
    </out>
</xsl:template>

<xsl:function name="h:getresource">
    <xsl:param name="component"/>
    <xsl:param name="item"/>
    <xsl:param name="params"/>
    <xsl:variable name="para" select="xfn:tokenize($params, ' ')"/>
    <xsl:variable name="xpath-str"><xsl:value-of select="concat('.', $component)"/></xsl:variable>
    <res>
        <xsl:for-each select="$resource/res">
            <xsl:variable name="e" as="item()*">
              <xsl:evaluate xpath="$xpath-str" context-item=".">
                 <xsl:with-param name="p1" select="$para[1]"/>
                 <xsl:with-param name="p2" select="$para[2]"/>
                 <xsl:with-param name="p3" select="$para[3]"/>
                 <xsl:with-param name="p4" select="$para[4]"/>
                 <xsl:with-param name="p5" select="$para[5]"/>
                 <xsl:with-param name="p6" select="$para[6]"/>
                 <xsl:with-param name="p7" select="$para[7]"/>
                 <xsl:with-param name="p8" select="$para[8]"/>
                 <xsl:with-param name="p9" select="$para[9]"/>
               </xsl:evaluate>
            </xsl:variable>
                 
            <xsl:for-each select="$e">
                 
                <xsl:choose>
                    <xsl:when test="string-length($item)">
                        <item><xsl:evaluate xpath="$item" context-item="."/></item>
                    </xsl:when>
                    <xsl:otherwise><xsl:copy-of select="."/></xsl:otherwise>
                </xsl:choose>
            </xsl:for-each>
        </xsl:for-each>
    </res>
</xsl:function>

</xsl:stylesheet>
