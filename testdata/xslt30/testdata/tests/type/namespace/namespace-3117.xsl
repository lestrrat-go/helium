<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns="http://base.test/">

<?spec xslt#xsl-element?>
    <!-- Purpose: Request prefix that is already mapped to requested NS, default set globally. -->

<xsl:template match = "/">
  <out xmlns:p1="http://xyz/">
    <xsl:element namespace="http://xyz/" name="p1:foo">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>