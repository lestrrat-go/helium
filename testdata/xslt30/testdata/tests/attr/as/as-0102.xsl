<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test with xsl:template and @as=empty-sequence(), the sequence constructor contains 
  				a sequence of a explicitly constructed built-in atomic types with empty sequence as 
  				an argument.Types constructed are:	xs:string, xs:boolean, xs:double, xs:dayTimeDuration,
  				xs:date -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:template match="/doc">
      <out>
         <xslt:call-template name="first"/>
         <xslt:call-template name="second"/>
         <xslt:call-template name="third"/>
         <xslt:call-template name="fourth"/>
         <xslt:call-template name="fifth"/>
      </out>
   </xslt:template>

   <xslt:template name="first" as="empty-sequence()">
	     <xslt:sequence select="xs:string(())"/>
   </xslt:template>

   <xslt:template name="second" as="empty-sequence()">
	     <xslt:sequence select="xs:boolean(())"/>
   </xslt:template>

   <xslt:template name="third" as="empty-sequence()">
	     <xslt:sequence select="xs:double(())"/>
   </xslt:template>

   <xslt:template name="fourth" as="empty-sequence()">
	     <xslt:sequence select="xs:dayTimeDuration(())"/>
   </xslt:template>

   <xslt:template name="fifth" as="empty-sequence()">
	     <xslt:sequence select="xs:date(())"/>
   </xslt:template>
</xslt:transform>
