<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">

   <xslt:output method="xml"/>
   <!-- Purpose: Test to verify that a parameter that has no 'select' attribute but has an 'as' 
  				attribute is carried automatically through built-in rules and type is retained.-->

   <xslt:template match="/">
      <out>
         <xslt:apply-templates>
    	       <xslt:with-param name="empty_string"/>
    	       <xslt:with-param name="empty_seq" as="xs:string?"/>
         </xslt:apply-templates>
      </out>
   </xslt:template>

   <xslt:template match="empty">
	     <xslt:param name="empty_string" select="'false'"/>
	     <xslt:param name="empty_seq" select="'10'"/>
	empty-CORRECT!
	Param:<xslt:value-of select="$empty_string eq ''"/>
	Param:<xslt:value-of select="empty($empty_seq)"/>
   </xslt:template>

   <xslt:template match="sub-empty">
	Print this only once.
</xslt:template>

   <xslt:template match="comment()">
</xslt:template>

   <xslt:template match="processing-instruction()">
	processing-instruction-CORRECT!
</xslt:template>

   <xslt:template match="text()">
</xslt:template>

   <xslt:template match="@*">
	This should not Print
</xslt:template>
</xslt:transform>
