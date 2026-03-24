<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:p1="http://testguys.com/">

<?spec xslt#xsl-element?>
    <!-- Purpose: Issue prefixed name in current default namespace, rather than the one assigned to tha prefix at outer level -->
  <!-- NOTE: Processor developers could legitimately disagree about where the default name
     has to be set in the result. It must be in effect for yyy, but could be for foo
     as well. The spec doesn't address this point. -->

<xsl:template match = "/">
  <out>
    <xsl:element name="p1:foo" namespace="http://other.com/" xmlns="http://other.com/">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>