<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:x="http://xxx.com/" xmlns:xs="http://www.w3.org/2001/XMLSchema">

    <xsl:param name="new-each-time" as="xs:string" static="yes" />

    <!-- deterministic function -->

    <xsl:function name="x:test" new-each-time="no" _new-each-time="{$new-each-time}" as="xs:string">
        <xsl:param name="n" as="xs:integer" />
        <xsl:variable name="new-node" as="element()">
            <e>
                <xsl:value-of select="$n" />
            </e>
        </xsl:variable>
        <xsl:sequence select="generate-id($new-node)" />
    </xsl:function>

    <xsl:template match="/">
        <xsl:variable name="ids" as="xs:string*">
            <xsl:for-each
                select="(1,4,6,8,3,5,6,2,1,3)">
                <xsl:sequence select="x:test(.)" />
            </xsl:for-each>
        </xsl:variable>
        <out>
            <xsl:value-of select="count(distinct-values($ids))" />
        </out>
    </xsl:template>


</xsl:stylesheet>
