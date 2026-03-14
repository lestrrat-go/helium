<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    expand-text="yes"
    exclude-result-prefixes="#all" 
    version="3.0">
    
 <xsl:template name="xsl:initial-template">
     <xsl:source-document href="sf-xml-to-json-A.xml" streamable="true">
         <json>{xml-to-json(.)}</json>
     </xsl:source-document>
 </xsl:template>
    
</xsl:stylesheet>