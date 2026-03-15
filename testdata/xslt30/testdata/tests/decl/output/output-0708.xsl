<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns="http://www.w3.org/1999/xhtml" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   version="3.0">
   
   <!-- JSON output: use a character map to suppress escaping of "/" -->

   <xsl:output method="json" use-character-maps="no-escape-slash" build-tree="no"/>
   
   <xsl:character-map name="no-escape-slash">
      <xsl:output-character character="/" string="/"/>
   </xsl:character-map>

   <xsl:template name="xsl:initial-template">
      <xsl:map>
         <xsl:map-entry key="'a'" select="'http://www.w3.org/2001/XMLSchema'"/>
      </xsl:map>
   </xsl:template>
</xsl:transform>
