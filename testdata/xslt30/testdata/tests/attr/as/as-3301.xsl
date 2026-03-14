<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://www.example.com/ns/notation-id-ref"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="my xs"
                version="2.0">
<!-- Purpose: Test type of global xsl:variable selecting a typed attribute node from input file and 
  				@as="attribute (QName, xs:NOTATION)".-->

   <xslt:import-schema namespace="http://www.example.com/ns/notation-id-ref"
                       schema-location="id-idref-notation-schemaas.xsd"/>

   <xslt:variable name="var1"
                  select="/my:docid/notation-elem/@my:filetype"
                  as="attribute(my:filetype, xs:NOTATION)"/>

   <xslt:template match="/my:docid">
      <out>
         <var1>
            <xslt:value-of select="$var1 instance of attribute(my:filetype, xs:NOTATION)"/>
            <xslt:value-of select="notation-elem/@my:filetype instance of attribute(my:filetype, xs:NOTATION)"/>
         </var1>
      </out>
   </xslt:template>
</xslt:transform>
