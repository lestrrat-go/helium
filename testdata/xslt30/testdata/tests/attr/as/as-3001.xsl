<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/union-list" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test type of global xsl:variable with @select selecting a typed element node 
  				from input file, @as="element(*, user-defined-list-type)". Item types in the 
  				list are: - built-in primitive, built-in derived, user-defined atomic, union-->

   <xslt:import-schema namespace="http://www.example.com/ns/union-list"
      schema-location="list-unionSchemaAs.xsd"/>

   <xslt:variable name="var1" select="/my:userNode/my:list-builtin" as="element(*, xs:NMTOKENS)"/>

   <xslt:variable name="var2" select="/my:userNode/my:list-primitive"
      as="element(*, my:myPrimitiveListType)"/>

   <xslt:variable name="var3" select="/my:userNode/my:list-derived"
      as="element(*, my:myDerivedListType)"/>

   <xslt:variable name="var4" select="/my:userNode/my:list-user-defined"
      as="element(*, my:myListType)"/>

   <xslt:variable name="var5" select="/my:userNode/my:list-union"
      as="element(*, my:myUnionListType)"/>

   <xslt:template match="/">
      <out>
         <var1>
            <xslt:value-of select="$var1 instance of element(*, xs:NMTOKENS)"/>
         </var1>
         <var2>
            <xslt:value-of select="$var2 instance of element(*, my:myPrimitiveListType)"/>
         </var2>
         <var3>
            <xslt:value-of select="$var3 instance of element(*, my:myDerivedListType)"/>
         </var3>
         <var4>
            <xslt:value-of select="$var4 instance of element(*, my:myListType)"/>
         </var4>
         <var5>
            <xslt:value-of select="$var5 instance of element(*, my:myUnionListType)"/>
         </var5>
      </out>
   </xslt:template>
</xslt:transform>
