<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0">
<!-- Purpose: Simple test for constructing a sequence from source -->

   <xslt:variable name="values" as="xs:string*">
      <xslt:for-each select="//a">
         <xslt:sequence select="."/>
      </xslt:for-each>
   </xslt:variable>

   <xslt:template match="/doc">
      <out>
         <xslt:value-of select="$values" separator="|"/>
      </out>
   </xslt:template>
</xslt:transform>
