<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test type of global xsl:variable where @select contains an xs:NMTOKEN, @as="xs:token".(subtype substitution)-->

   <xslt:variable name="var1" select="/doc-schemaas/elem-NMTOKEN" as="xs:token"/>

   <xslt:template match="/doc-schemaas">
      <out>
         <var1>
            <xslt:value-of select="$var1 instance of xs:NMTOKEN"/>
            <xslt:value-of select="$var1 instance of xs:token"/>
         </var1>
      </out>
   </xslt:template>
</xslt:transform>
