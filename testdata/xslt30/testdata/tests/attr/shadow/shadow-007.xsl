<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

   <!-- underscored attributes on LREs are unaffected -->
    
   <xsl:template name="main">
      <out one="1" _one="1.0" _two="two"/>
   </xsl:template>
   
 
   
</xsl:transform>
