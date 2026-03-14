<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <?spec xslt#attribute-sets?>
<!-- test a composite attribute set where both parts allocate local variables
     (to test allocation of stack space for these variables) -->

   <xsl:attribute-set name="b" >
      <xsl:attribute name="sum-sq" select="sum(for $i in 1 to 5 return $i*$i)"/>
   </xsl:attribute-set> 

   <xsl:attribute-set name="b">
     <xsl:attribute name="avg-sq" select="avg(for $i in 1 to 5 return $i*$i)"/>
   </xsl:attribute-set> 
   
   <xsl:template match="/">
     <a xsl:use-attribute-sets="b"/>
   </xsl:template>

</xsl:transform>

