<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
    <!-- Purpose: Test what happens when namespace attrib is a known URI, but no prefixes involved. -->

<xsl:template match = "/">
  <out xmlns="http://abc/">
    <xsl:element namespace="http://abc/" name="foo">
      <xsl:element name="yyy"/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>


