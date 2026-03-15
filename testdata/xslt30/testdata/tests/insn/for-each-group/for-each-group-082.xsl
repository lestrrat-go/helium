<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet exclude-result-prefixes="xs" version="2.0" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
  <!-- group elements by local-name -->
  
  <xsl:output encoding="UTF-8" method="xml"/>
  
  <xsl:variable name="in">
    <root xmlns:a="http://example.com" xmlns:b="http://example.com">
      <line att="1"/>
      <line att="2"/>
      <line2 att="3"/>
      <line att="4"/>
      <line3 att="5"/>
      <line3 att="6"/>
      <line att="7"/>
      <line att="8"/>
      <line3 att="9"/>
      <line3 att="10"/>
      <line att="11"/>
      <a:line att="12"/>
      <a:line att="13"/>
      <a:line3 att="14"/>
      <b:line3 att="15"/>
      <b:line att="16"/>
    </root>
  </xsl:variable>
  
  <xsl:template name="main">
    <xsl:apply-templates select="$in"/>
  </xsl:template>
  
  <xsl:template match="root">
    <out>
      <xsl:for-each-group select="*" group-by="node-name(.)">
        <group>
          <xsl:copy-of select="current-group()" copy-namespaces="no"/>
        </group>
      </xsl:for-each-group>
    </out>
  </xsl:template>
</xsl:stylesheet>