<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test that an error is raised if @as is used 
  				in context where it is not permitted (@as used 
  				on xsl:call-template instruction). -->

   <xslt:template match="/">
      <xslt:call-template name="temp" as="xs:integer"/>
   </xslt:template>

   <xslt:template name="temp">5</xslt:template>
   
</xslt:transform>
