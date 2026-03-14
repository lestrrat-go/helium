<?xml version="1.0" encoding="ISO-8859-1" ?>
<xsl:transform 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  version="2.0">
  
  <!-- Modified example taken from the example "Conflicting Namespace Prefixes" in section 11.7 of the spec -->
  <!-- This should not generate an error. The prefix "p" is not available for use by namespace fixup, so the
        element must be allocated a different prefix  -->
  <?spec xslt#creating-namespace-nodes?>

  <xsl:template match="/">
    <xsl:variable name="e" as="element()">
    <xsl:element name="p:item" xmlns:p="http://p.uri/">
       <xsl:namespace name="p">http://q.uri/</xsl:namespace>

    </xsl:element>
    </xsl:variable>
    <ok><xsl:value-of select="not(starts-with(name($e), 'p:'))"/></ok>
  </xsl:template>
  
</xsl:transform>

