<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">

   <xslt:output method="xml"/>
   <!-- Purpose: Test to verify that the value of parameter with a sequence 
  				constructor is carried automatically through built-in rules 
  				to explicit templates for descendants.-->

   <xslt:template match="/">
      <out>
         <xslt:apply-templates>
    
    	       <xslt:with-param name="first_seq">
    		         <param>
                  <temp>Temporaty tree</temp>
               </param>
    	       </xslt:with-param>
    	
    	       <xslt:with-param name="second_seq" as="xs:string">
    		         <param>
                  <temp>Temporaty tree</temp>
               </param>
    	       </xslt:with-param>
    	
         </xslt:apply-templates>
      </out>
   </xslt:template>

   <xslt:template match="empty">
	     <xslt:param name="first_seq">abc</xslt:param>
	     <xslt:param name="second_seq">def</xslt:param>
	empty-CORRECT!
	Param1:<xslt:value-of select="$first_seq"/>
	Param2:<xslt:value-of select="$second_seq"/>
	Param1 is string:<xslt:value-of select="$first_seq instance of xs:string"/>
	Param2 is string:<xslt:value-of select="$second_seq instance of xs:string"/>
	Param1 nodes:<xslt:value-of select="$first_seq//name()"/>
   </xslt:template>

   <xslt:template match="sub-empty">
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
