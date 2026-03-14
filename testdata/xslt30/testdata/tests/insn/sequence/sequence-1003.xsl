<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0">
<!-- Purpose: Test for a sequence in a global parameter -->

   <xslt:param name="values" as="xs:string*">
      <xslt:sequence select="('a','b','c')"/>
   </xslt:param>

   <xslt:template match="/doc">
      <out>
         <xslt:value-of select="$values" separator="|"/>
      </out>
   </xslt:template>
</xslt:transform>
