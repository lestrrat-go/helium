<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://www.example.com/ns/various"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                version="2.0">
<!-- Purpose: Test type of global xsl:variable with @as="element(*, user-defined-atomic-type)+". 
  				@select has: - a typed element node from input file, a sequence of typed element nodes from input file -->

   <xslt:import-schema namespace="http://www.example.com/ns/various"
                       schema-location="variousTypesSchemaAs.xsd"/>

   <xslt:variable name="var1"
                  select="/my:userNode/my:simpleUserElem[1]"
                  as="element(*, my:partNumberType)+"/>

   <xslt:variable name="var2"
                  select="/my:userNode/my:simpleUserElem"
                  as="element(*, my:partNumberType)+"/>

   <xslt:template match="/my:userNode">
      <out>
         <var1>
            <xslt:value-of select="$var1 instance of element(*, my:partNumberType)+"/>
            <xslt:value-of select="$var1 instance of element(*, my:partNumberType)"/>
         </var1>
         <var2>
            <xslt:value-of select="$var2 instance of element(*, my:partNumberType)+"/>
            <xslt:value-of select="$var2 instance of element(*, my:partNumberType)"/>
         </var2>
      </out>
   </xslt:template>
</xslt:transform>
