<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test type of global xsl:variable where @select contains an xs:ID, @as="xs:normalizedString".(subtype substitution)-->

   <xslt:variable name="var1" select="/doc-schemaas/elem-ID-attr/@attr1" as="xs:normalizedString"/>

   <xslt:template match="/doc-schemaas">
      <out>
         <var1>
            <xslt:value-of select="$var1 instance of xs:ID"/>
            <xslt:value-of select="$var1 instance of xs:normalizedString"/>
         </var1>
      </out>
   </xslt:template>
</xslt:transform>
