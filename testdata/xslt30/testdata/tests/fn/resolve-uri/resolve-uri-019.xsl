<?xml version="1.0"?>
<?spec fo#func-resolve-uri?>
<?error ?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

    <!-- Test the resolve-uri() function with invalid base URI -->

<xsl:output method="xml" indent="yes"/>

<xsl:template match="doc" xml:base="http://no.other/home/">
  <out>
    <a><xsl:value-of select="resolve-uri('some.uri', 'http://localhost/base/##frag')"/></a>       
  </out>
</xsl:template>

</xsl:stylesheet>
