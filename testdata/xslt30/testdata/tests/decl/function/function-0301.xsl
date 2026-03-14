<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>

<xsl:stylesheet version="2.0" 
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                 xmlns:sax="http://icl.com/saxon"
                 xmlns:xyz="http://foobar/"
                 exclude-result-prefixes="sax xyz">
                 
<!-- stylesheet function invoked from within sort key -->
   
<xsl:function name="xyz:key">
    <xsl:param name="node" as="node()"/>
    <xsl:sequence select="concat(string(string-length($node/@c)), '-', $node/@c)"/>
</xsl:function>
   
<xsl:template match="/">
    <out>
        <xsl:for-each select="//a"><xsl:sort select="xyz:key(.)"/>
           <item><xsl:value-of select="xyz:key(.)"/>: <xsl:value-of select="@c"/></item>;
        </xsl:for-each>   
    </out>
</xsl:template>

</xsl:stylesheet>

