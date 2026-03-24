<?xml version="1.0"?>
<?spec fo#func-resolve-uri?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

    <!-- Test the resolve-uri() function with one argument -->


<xsl:template match="doc" xml:base="http://some.other/home/">
  <out>
    <a><xsl:value-of select="resolve-uri('')"/></a>
    <a><xsl:value-of select="resolve-uri('banana.xml')"/></a>    
    <a><xsl:value-of select="resolve-uri('..')"/></a>     
    <a><xsl:value-of select="resolve-uri('http://www.wrox.com/')"/></a>            
  </out>
</xsl:template>

</xsl:stylesheet>
