<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

   <!-- underscored attributes on LREs are unaffected -->
    
   <xsl:template name="main" default-collation="http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive">
      <out xsl:_default-collation="http://www.w3.org/2005/xpath-functions/collation/codepoint">
         <xsl:value-of select="'this' = 'THIS'"/>
      </out>
   </xsl:template>
   
 
   
</xsl:transform>
