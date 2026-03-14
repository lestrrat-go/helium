<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:pfix="http://party.com/">

<?spec xslt#creating-attributes?>
    <!-- Purpose: Declare prefixed namespace at stylesheet level, then use another prefix in
    attribute name, but same namespace. New namespace-decl must be issued, or change prefix. -->
  <!-- This test raises another facet of the issue of supplied prefixes on the attribute
    name. Processor developers could disagree about whether the combination of a prefixed
    name and an explicit namespace attribute with a URI signals a request from the
    stylesheet to generate a namespace declaration, even when the requested URI is already
    available to apply to the attribute simply by using a different prefix. In other words,
    does the stylesheet really want to ensure that an xmlns:other declaration is issued? -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="/">
  <out>
    <xsl:element name="inner">
      <xsl:attribute name="other:nuts" namespace="http://party.com/">almond</xsl:attribute>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>