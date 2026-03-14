<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns="http://testguys.com/">

<?spec xslt#xsl-element?>
    <!-- Purpose: Reset default locally, but namespace attribute matches stylesheet default. -->
  <!-- NOTE: Processor developers could legitimately disagree about where the default name
     has to be set the 2nd time in the result. It must be correct for yyy, but could be reset for foo
     as well. The spec doesn't address this point. -->

<xsl:template match = "/">
  <out>
    <xsl:element name="q:foo" namespace="http://testguys.com/" xmlns="http://other.com/">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>