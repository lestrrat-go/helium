<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://www.example.com/ns/match/var"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" 
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   xmlns:f="http://local.functions/"
   exclude-result-prefixes="my xs f" version="2.0">
   <!-- Purpose: Test of template matching with schema-attribute($name) with an explicitly constructed 
  				attribute node without a parent and validated with @validation=strict. -->
   
   <!-- Copied from match-193, but using a local variable rather than a global variable. Saxon bug 5732 -->

   <xslt:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>
   
   <xslt:function name="f:orphan-attribute" as="schema-attribute(my:specialPart)">
      <xslt:param name="content" as="xs:string*"/>
      <xslt:for-each select="year-from-date(current-date()) to year-from-date(current-date()) + 1 - 1">
         <xslt:attribute name="my:specialPart" validation="strict"><xslt:value-of select="$content || 'TM'" separator="{.}"/></xslt:attribute>
      </xslt:for-each>
   </xslt:function>

   <xslt:template match="/">
      <out>
         <A><xslt:value-of select="f:orphan-attribute('000-') => substring(1, 6)"/></A>       
      </out>
   </xslt:template>

 
</xslt:transform>
