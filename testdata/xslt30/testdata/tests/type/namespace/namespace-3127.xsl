<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
    <!-- Purpose: Prefixed xmlns declaration and same-prefixed name; namespace matches default set locally. -->
  <!-- NOTE: Processor developers could legitimately disagree about where the default name
     has to be reset in the result. It must be in effect for yyy, but could be for foo
     as well. The spec doesn't address this point.
     On the other hand, p2 must be set to "barz.com" for the yyy element. -->

<xsl:template match = "/">
  <out>
    <xsl:element name="p2:foo" namespace="http://other.com/" xmlns="http://other.com/" xmlns:p2="http://barz.com/">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>