<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test type of global xsl:variable with @select selecting a typed element node from input file, 
  				@as="schema-element($QName)". Verify that selected element is annotated as an instance of the 
  				type defined by the schema element declaration $QName and its name= $QName.-->

   <xslt:import-schema schema-location="builtinTypeSchemaAs.xsd"/>

   <xslt:variable name="var1"
                  select="/doc-schemaas/elem-anyURI"
                  as="schema-element(elem-anyURI)"/>

   <xslt:template match="/">
      <out>
         <var1>
            <xslt:value-of select="$var1 instance of element(elem-anyURI, xs:anyURI)"/>
            <xslt:value-of select="$var1 instance of schema-element(elem-anyURI)"/>
            <xslt:value-of select="doc-schemaas/elem-anyURI instance of schema-element(elem-anyURI)"/>
         </var1>
      </out>
   </xslt:template>
</xslt:transform>
