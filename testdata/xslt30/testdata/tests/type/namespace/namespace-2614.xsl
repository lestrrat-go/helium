<?xml version="1.0" encoding="ISO-8859-1" ?>
<xsl:transform 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  version="2.0">
  
  <!-- Modified example taken from the example "Confilicting Namespace Prefixes" in section 11.7 of the spec -->
  <!-- exclude-result-prefixes added. -->
  <!-- This should not generate an error (namespace fixup should invent a prefix) -->
  <?spec xslt#creating-namespace-nodes?>

  <xsl:template match="/">
    <p:item xmlns:p="http://p.uri/" xsl:exclude-result-prefixes="p"><xsl:namespace name="p">http://q.uri/</xsl:namespace></p:item>
  </xsl:template>
  
</xsl:transform>
