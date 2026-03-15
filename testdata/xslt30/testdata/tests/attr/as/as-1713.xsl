<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test with xsl:template that contains an xs:NMTOKEN, 
  				@as="xs:token".(subtype substitution)-->

   <xslt:template match="/">
      <out>
         <temp1>
            <xslt:call-template name="temp1"/>
         </temp1>
      </out>
   </xslt:template>

   <xslt:template name="temp1" as="xs:token">
      <xslt:value-of select="/doc-schemaas/elem-NMTOKEN"/>
   </xslt:template>
</xslt:transform>
