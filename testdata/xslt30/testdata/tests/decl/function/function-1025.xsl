<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:x="http://xxx.com/" xmlns:xs="http://www.w3.org/2001/XMLSchema">

    <xsl:param name="new-each-time" as="xs:string" static="yes" />

    <!-- deterministic function -->

    <xsl:function name="x:test" as="element()" new-each-time="no" _new-each-time="{$new-each-time}">
        <xsl:param name="n" as="xs:integer" />
        <e>
            <xsl:value-of select="$n" />
        </e>
    </xsl:function>

    <xsl:template match="/">
        <xsl:variable name="nodes" as="element()*">
            <xsl:for-each
                select="(1,4,6,8,3,5,6,2,1,3)">
                <xsl:sequence select="x:test(.)" />
            </xsl:for-each>
        </xsl:variable>
        <out>
            <xsl:value-of select="count($nodes | ())" />
        </out>
    </xsl:template>


</xsl:stylesheet>
