<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet exclude-result-prefixes="xs" version="2.0" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xpath-default-namespace="http://www.example.com/">
  
  <xsl:variable name="data">
    <doc xmlns="http://www.example.com/">
      <item><key>A</key><value>10</value></item>
      <item><key>A</key><value>20</value></item>
      <item><key>B</key><value>30</value></item>
      <item><key>B</key><value>40</value></item>
    </doc>
  </xsl:variable>
  
  <xsl:template name="xsl:initial-template">
    <out>
      <xsl:for-each-group select="$data//item" group-by="key">
        <group key="{current-grouping-key()}" total="{sum(current-group()/value)}"/>
      </xsl:for-each-group>
    </out>
   </xsl:template>
  
</xsl:stylesheet>