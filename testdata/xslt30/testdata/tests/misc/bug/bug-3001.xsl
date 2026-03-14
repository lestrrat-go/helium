<?xml version="1.0"?>


<?spec xpath#id-predicates?>
<!-- BUG: Saxon -->
<!-- DESCRIPTION: -->

<!-- from http://www.zvon.org/xxl/XSLTreference/OutputExamples/example_8_87_frame.html -->

<xsl:stylesheet xmlns:xsl = "http://www.w3.org/1999/XSL/Transform" version = "2.0" > 
  <xsl:output method = "text" /> 

  <xsl:template match = "/" > 
      <xsl:text >
      </xsl:text> 
               <xsl:apply-templates select = "//CCC[.='c21']/ancestor::*" mode = "print" /> 
               <xsl:text >
      </xsl:text> 
               <xsl:apply-templates select = "//DDD[1]/EEE[1]/ancestor::*" mode = "print" /> 
  </xsl:template> 

  <xsl:template match = "*" mode = "print" > 
       <xsl:value-of select = "concat(name(),'{',position(),'}')" /> 
       <xsl:if test = "not(position()=last())" > 
            <xsl:text > - </xsl:text> 
       </xsl:if> 
  </xsl:template> 
</xsl:stylesheet> 
