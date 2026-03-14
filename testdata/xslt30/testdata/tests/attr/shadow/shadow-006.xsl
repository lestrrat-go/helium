<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

   <!-- shadow attribute (static AVT) on @xpath-default-namespace -->
   
   <xsl:variable name="ONE" static="yes" select="'http://www.example.com/one'"/>
   
   <xsl:variable name="data">
      <e xmlns="http://www.example.com/one">1</e>
      <e xmlns="http://www.example.com/two">2</e>
   </xsl:variable>
   
   <xsl:template name="main" _xpath-default-namespace="{$ONE}">
      <out><xsl:value-of select="$data/e"/></out>
   </xsl:template>
   
 
   
</xsl:transform>
