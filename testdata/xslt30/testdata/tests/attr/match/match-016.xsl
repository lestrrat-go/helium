<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
    <!-- Purpose: Test of large union. -->

<xsl:template match="doc">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="aac|llh|oop|aah|ssb|iii|rre|eek|xxo|aar|sst|bbd|eeo|xxi|ddg|nne">
  <xsl:value-of select="."/><xsl:text>
</xsl:text>
</xsl:template>

<xsl:template match="text()"/>

</xsl:stylesheet>
