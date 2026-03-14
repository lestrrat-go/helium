<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet exclude-result-prefixes="#all" version="3.0" xmlns:x="http://example.com/x"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
 
  <xsl:template as="element(x:description)" name="xsl:initial-template">
    <xsl:element inherit-namespaces="no" name="description" namespace="http://example.com/x">
      <x:scenario>
        <e/>
      </x:scenario>
    </xsl:element>
  </xsl:template>
  
</xsl:stylesheet>