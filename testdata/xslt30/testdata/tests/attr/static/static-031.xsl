<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
   xmlns:xs="http://www.w3.org/2001/XMLSchema" 
   exclude-result-prefixes="#all"
   version="3.0">

   <!-- A static parameter whose type has a cardinality indicator (Saxon-JS bug 4619) -->
   
   <xsl:param name="static-param" static="yes" as="xs:string?" select="'Hello world'"/>
   
   <xsl:template name="xsl:initial-template">
      <result value="{$static-param}"/>
   </xsl:template>
   
</xsl:transform>
