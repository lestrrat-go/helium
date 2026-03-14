<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns="http://testguys.com">

<?spec xslt#creating-attributes?>
    <!-- Purpose: Test for namespace attribute with new URI and prefix requested. -->

<xsl:template match = "/">
  <out>
    <xsl:element name="q:foo" namespace="http://other.com">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>


