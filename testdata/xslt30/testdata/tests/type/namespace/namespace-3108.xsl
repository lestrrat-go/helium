<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
    <!-- Purpose: Use prefixed xmlns declaration, namespace attrib to same URI, different prefix on name. -->
  <!-- NOTE: Processor developers could legitimately disagree about where the "p2" name
     has to be in scope on the result. It must be in effect for yyy, but could be for foo
     as well. The spec doesn't address this point. -->

<xsl:template match = "/">
  <out>
    <xsl:element name="az:foo" namespace="http://barz.com/" xmlns:p2="http://barz.com/">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>