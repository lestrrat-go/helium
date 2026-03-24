<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs"
  expand-text="yes">
   <!-- Purpose: See that match="$nodeset" gives acceptable performance -->
  
   <xsl:mode on-no-match="deep-skip"/>
  
   <xsl:param name="count" select="1000" as="xs:integer"/>

   <xsl:variable name="nodes" as="element()*">
     <xsl:for-each select="1 to $count">
       <a>A{.}</a>
       <b>B{.}</b>
     </xsl:for-each>
   </xsl:variable>
  
   <xsl:variable name="a-nodes" select="$nodes[self::a]"/>
  
   <xsl:template name="xsl:initial-template">
     <out>
        <xsl:variable name="numbers" as="xs:integer*">
          <xsl:apply-templates select="$nodes"/>
        </xsl:variable>
        <xsl:value-of select="sum($numbers)"/>
     </out>
   </xsl:template>
  
   <xsl:template match="$a-nodes">
      <xsl:sequence select="xs:integer(substring(., 2))"/>
   </xsl:template>
  
  
</xsl:transform>
