<?xml version="1.0"?> 

<!-- conflicting use-accumulator attribute on xsl:mode - with namespaces -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/acc-ns/p"
  xmlns:q="http://www.w3.org/acc-ns/q"
  exclude-result-prefixes="xs">

  <xsl:accumulator name="p:a" initial-value="0" as="xs:integer">
    <xsl:accumulator-rule match="*" select="$value+1"/>
  </xsl:accumulator>
  
  <xsl:accumulator name="q:b" initial-value="0" as="xs:integer">
    <xsl:accumulator-rule match="*" select="$value+2"/>
  </xsl:accumulator>
  
  <xsl:mode name="m" use-accumulators="p:a q:b"/>
  
    
</xsl:stylesheet>