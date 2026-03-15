<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
   
   <!--
      NOTE: test deliberately throws an error by hand, because it is not possible to test this in a processor-independent way
      
         A non-schema-aware processor
                     must raise a non-recoverable dynamic error if the input to the processor
                     includes a node with a type annotation
                     other than xs:untyped or xs:untypedAtomic, or an
                     atomic value of a type other than those which a basic XSLT processor
                     supports.-->
   <xsl:template name="main">
      <out>
         <xsl:assert error-code="Q{{http://www.w3.org/2005/xqt-errors}}XTDE1665" test="false()" />
         <x/>
      </out>
   </xsl:template>
</xsl:stylesheet>
