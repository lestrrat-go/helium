<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:my="http://example1.com"
  exclude-result-prefixes="my">
  
  <!-- Test with various attributes in xsl:result-document and an output declaration 
    that overrides another output declaration. Attributes that are being tested are:
    method, byte-order-mark, encoding, indent, omit-xml-declaration, 
    standalone, output-version, undeclare-prefixes -->
  
  <xsl:import href="result-document-0239a.xsl" />
  
  <xsl:output name="my:temp-output" method="text" encoding="UTF-8" byte-order-mark="no" indent="no" 
    omit-xml-declaration="yes" standalone="no" version="1.1" undeclare-prefixes="yes"/>
  
  <xsl:template match="doc">
    <xsl:result-document format="my:temp-output">
      <out>This document should be serialized as text, not as xml, the output definition used is not overriden.</out>
    </xsl:result-document>
  </xsl:template>
  
</xsl:stylesheet>
