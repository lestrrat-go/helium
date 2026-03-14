<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test that xsl:version is a valid system-property and the returned result is an xs:string with numeric value between 1.0 and 3.0.-->

   <xslt:template match="/">
      <out>
         <xslt:value-of select="system-property('xslt:version') instance of xs:string"/>
         <xslt:value-of select="xs:double(system-property('xslt:version')) ge 1.0 and xs:double(system-property('xslt:version')) le 3.0"/>
      </out>
   </xslt:template>
</xslt:transform>
