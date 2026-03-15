<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test with global xsl:variable where of @select has a value obtained at run time, 
  				@as="xs:anyAtomicType". Verify that the resulting variable type is xs:anyAtomicType.   -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:variable name="var1" select="/doc" as="xs:anyAtomicType"/>

   <xslt:variable name="var2" select="/doc/item" as="xs:anyAtomicType"/>

   <xslt:variable name="var3" select="/doc/item/@attr" as="xs:anyAtomicType"/>

   <xslt:template match="/doc">
      <out>
         <xslt:value-of select="$var1 instance of xs:anyAtomicType"/>
         <xslt:value-of select="$var2 instance of xs:anyAtomicType"/>
         <xslt:value-of select="$var3 instance of xs:anyAtomicType"/>
      </out>
   </xslt:template>
</xslt:transform>
