<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>

<xsl:stylesheet version="2.0" 
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                 xmlns:xyz="http://foobar/"
                 exclude-result-prefixes="xyz" >
                 
<!-- stylesheet function with 0 arguments -->                 
                 
<xsl:function name="xyz:abc">
   <xsl:sequence select="'x'"/>
</xsl:function>

<xsl:template match="/">
    <y><xsl:value-of select="xyz:abc()"/></y>
</xsl:template>

</xsl:stylesheet>

