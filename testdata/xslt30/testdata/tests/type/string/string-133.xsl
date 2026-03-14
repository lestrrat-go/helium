<?xml version="1.0" encoding="utf-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"

 xmlns:xs="http://www.w3.org/2001/XMLSchema-datatypes"
  xmlns:dp="http://dpawson.co.uk/ns"
                version="2.0">

 <xsl:template match="doc">
 <doc><xsl:apply-templates/></doc>
 </xsl:template>
 
 <xsl:template match="node[node]" priority="10">
   <xsl:variable name="theseChildren" select="node/@name"/>
   <a>
     <xsl:value-of select="string($theseChildren[1])"/>
   </a>
 </xsl:template>


</xsl:stylesheet>