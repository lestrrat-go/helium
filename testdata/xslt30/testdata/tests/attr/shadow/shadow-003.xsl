<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" _version="{system-property('xsl:version')}" version="1.0" id="shadow-003">

   <!-- shadow attribute (static AVT) on xsl:stylesheet/@version -->
   <!-- A 3.0 processor does NOT use backwards-compatible mode here -->
   

   <xsl:template name="main">
      <out><xsl:value-of select="1 to 5"/></out>
   </xsl:template>
   
</xsl:transform>
