<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test that an error is raised if in xsl:template @as=xs:double 
  				but the value in the sequence constructor cannot be converted 
  				to xs:double. -->

   <xslt:template match="/">
      <xslt:call-template name="temp"/>
   </xslt:template>

   <xslt:template name="temp" as="xs:double">hello</xslt:template>
   
</xslt:transform>
