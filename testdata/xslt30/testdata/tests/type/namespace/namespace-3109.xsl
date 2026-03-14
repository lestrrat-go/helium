<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns="http://testguys.com/">

<?spec xslt#xsl-element?>
    <!-- Purpose: Set default namespace, then reset via xmlns declaration in xsl:element. -->

<xsl:template match = "/">
  <out>
    <xsl:element name="foo" xmlns="">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>


