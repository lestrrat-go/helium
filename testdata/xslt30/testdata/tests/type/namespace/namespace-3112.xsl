<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns="http://testguys.com/">

<?spec xslt#xsl-element?>
    <!-- Purpose: Specify an empty namespace; default NS set at two levels. -->

<xsl:template match = "/">
  <out>
    <xsl:element namespace="" name="foo" xmlns="http://other.com/">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>


