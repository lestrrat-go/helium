<?xml version="1.0"?>
<?spec fo#func-base-uri?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

    <!-- Test the base-uri() function with no parameters -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="base-uri()"/>
  </out>
</xsl:template>

</xsl:stylesheet>
