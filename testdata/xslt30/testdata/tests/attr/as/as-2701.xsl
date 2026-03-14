<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test with a sequence of xs:ID values obtained at run time in @select 
  				of a local xsl:variable and @as=xs:ID+. Verify the variable is of type xs:ID+. -->

   <xslt:template match="/doc-schemaas">
      <xslt:variable name="var1" select="//@attr1" as="xs:ID+"/>
      <out>
         <xslt:value-of select="$var1"/>
         <xslt:text> * </xslt:text>
         <xslt:value-of select="$var1 instance of xs:ID"/>
         <xslt:value-of select="$var1 instance of xs:ID+"/>
      </out>
   </xslt:template>
</xslt:transform>
