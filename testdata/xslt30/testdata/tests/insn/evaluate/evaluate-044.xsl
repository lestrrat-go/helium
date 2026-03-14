<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xsd="http://www.w3.org/2001/XMLSchema"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:foo="http://rockspace.com/foo"
  xmlns:file="http://rockspace.com/file"
  xmlns:fox="http://rockspace.com/fox"
  exclude-result-prefixes="xsd foo"
  version="3.0">
  
  <xsl:function name="fox:a" as="xs:string?" visibility="public">
    <xsl:param name="context" as="xs:string"/>
    <xsl:sequence select="$context"/>
  </xsl:function>
  
  <xsl:template name="go">
    <out>
      <xsl:variable name="fileNS" as="element()">
        <file:file fox:fox=""/>
      </xsl:variable>
      <xsl:evaluate xpath="'''string''! fox:a(.)'" context-item="$fileNS" namespace-context="$fileNS"/>
    </out>
  </xsl:template>
  
  
</xsl:stylesheet>
