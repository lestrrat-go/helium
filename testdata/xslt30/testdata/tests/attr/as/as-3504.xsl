<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test of xsl:template with @as="schema-element($QName) *". 
  				Sequence constructor contains: -(), a typed element node 
  				from input file, a set of typed element nodes from input file-->

   <xslt:import-schema namespace="http://www.example.com/ns/various"
      schema-location="variousTypesSchemaAs.xsd"/>

   <xslt:template match="/">
      <out>
         <temp1>
            <xslt:call-template name="temp1"/>
         </temp1>
         <temp2>
            <xslt:call-template name="temp2"/>
         </temp2>
         <temp3>
            <xslt:call-template name="temp3"/>
         </temp3>
      </out>
   </xslt:template>

   <xslt:template name="temp1" as="schema-element(my:simpleUserElem)*"> </xslt:template>

   <xslt:template name="temp2" as="schema-element(my:simpleUserElem)*">
      <xslt:copy-of select="my:userNode/my:simpleUserElem[1]" validation="strict"/>
   </xslt:template>

   <xslt:template name="temp3" as="schema-element(my:simpleUserElem)*">
      <xslt:copy-of select="my:userNode/my:simpleUserElem" validation="strict"/>
   </xslt:template>
</xslt:transform>
