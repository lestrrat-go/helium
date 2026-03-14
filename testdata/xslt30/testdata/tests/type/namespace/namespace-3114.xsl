<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
    <!-- Purpose: Set default namespace at two levels, no prefixes involved. -->

<xsl:template match = "/">
  <out xmlns="http://abc/">
    <xsl:element namespace="http://abc/" name="foo" xmlns="http://other.com/">
      <xsl:element name="yyy"/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>


