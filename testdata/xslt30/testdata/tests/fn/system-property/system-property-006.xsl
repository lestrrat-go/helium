<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test that xsl:is-schema-aware is a valid system-property and the returned result is an xs:string - one of 'yes' or 'no'.-->

   <xslt:template match="/">
      <out>
         <xslt:value-of select="system-property('xslt:is-schema-aware') instance of xs:string"/>
         <xslt:value-of select="system-property('xslt:is-schema-aware')='yes' or system-property('xslt:is-schema-aware')='no'"/>
      </out>
   </xslt:template>
</xslt:transform>
