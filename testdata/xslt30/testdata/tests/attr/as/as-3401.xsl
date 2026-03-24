<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://www.example.com/ns/notation-id-ref"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="my xs" version="2.0">
   <!-- Purpose: Test type of global xsl:variable selecting a typed attribute node from 
  				input file and @as="attribute(QName, built-in-list-type)".
  				Types tested: xs:NMTOKENS, xs:ENTITIES xs:IDREFS.-->

   <xslt:import-schema namespace="http://www.example.com/ns/notation-id-ref"
      schema-location="id-idref-notation-schemaas.xsd"/>

   <xslt:variable name="var1" select="/my:docid/nmtoken-elem/@my:toks"
      as="attribute(my:toks, xs:NMTOKENS)"/>

   <xslt:variable name="var2" select="/my:docid/entity-elem/@my:entities"
      as="attribute(my:entities, xs:ENTITIES)"/>

   <xslt:variable name="var3" select="/my:docid/idrefs-elem/@my:refs"
      as="attribute(my:refs, xs:IDREFS)"/>

   <xslt:template match="/my:docid">
      <out>
         <var1>
            <xslt:value-of select="$var1 instance of attribute(my:toks, xs:NMTOKENS)"/>
            <xslt:value-of
               select="nmtoken-elem/@my:toks instance of attribute(my:toks, xs:NMTOKENS)"/>
         </var1>
         <var2>
            <xslt:value-of select="$var2 instance of attribute(my:entities, xs:ENTITIES)"/>
            <xslt:value-of
               select="entity-elem/@my:entities instance of attribute(my:entities, xs:ENTITIES)"/>
         </var2>
         <var3>
            <xslt:value-of select="$var3 instance of attribute(my:refs, xs:IDREFS)"/>
            <xslt:value-of select="idrefs-elem/@my:refs instance of attribute(my:refs, xs:IDREFS)"/>
         </var3>
      </out>
   </xslt:template>
</xslt:transform>
