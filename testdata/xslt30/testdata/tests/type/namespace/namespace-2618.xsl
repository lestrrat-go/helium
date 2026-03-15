<?xml version="1.0" encoding="ISO-8859-1" ?>
<xsl:transform 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  version="2.0">
  
  <!-- Incorrect example taken from the example "Confilicting Namespace Prefixes" in section 11.7 of the spec -->
  <!-- This should generate an error (two conflicting namespace nodes). Corrected versions in nspc70 and nspc71 -->
  <?spec xslt#creating-namespace-nodes?>
  <?error ?>

  <xsl:template match="/">
    <p:item xmlns:p="http://p.uri/"><xsl:namespace name="p">http://q.uri/</xsl:namespace></p:item>
  </xsl:template>
  
</xsl:transform>
