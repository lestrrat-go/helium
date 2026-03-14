<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns="http://testguys.com">

<?spec xslt#xsl-element?>
  <!-- Section: 7.1.2 Creating Elements -->
    <!-- Purpose: Test for namespace attribute matching stylesheet default. -->

<xsl:template match = "/">
  <out>
    <xsl:element name="q:foo" namespace="http://testguys.com">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>


