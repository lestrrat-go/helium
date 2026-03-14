<?xml version="1.0"?>
<?spec fo#func-resolve-uri?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

    <!-- Test the resolve-uri() function with base URI being a relative URI:
         no longer an error in April 2005 draft -->
    <!-- But it becomes an error (again!) as a result of erratum FO.E1 -->     

<xsl:output method="xml" indent="yes"/>

<xsl:template match="doc" xml:base="http://no.other/home/">
  <out>
    <a><xsl:value-of select="resolve-uri('index.html', '/html/base/')"/></a>       
  </out>
</xsl:template>

</xsl:stylesheet>
