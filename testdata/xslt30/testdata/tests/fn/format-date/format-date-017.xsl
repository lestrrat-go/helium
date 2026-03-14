<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec xslt#format-date?>
  <!-- PURPOSE: XSLT 2.0: test format-time(): timezones in GMT+x notation -->
  
  <!-- DISCUSSION (Michael Kay 2019-12-09)
    
    The expected output of this test looks questionable both under the XSLT 2.0 rules and under
    the Functions & Operators 3.1 rules. 
    
    For XSLT 2.0, Erratum E29 says:
    
    "For a timezone offset (component specifier z), the full representation consists of a sign for 
    the offset, the number of hours of the offset, and if the offset is not an integral number of 
    hours, a colon (:) followed by the two digits of the minutes of the offset.". But that can be
    read as producing "GMT-5" in preference to "GMT-05" which this test expects.
    
    For XSLT 3.0 (or more specifically F+O 3.1), arguably section 9.8.4.6 doesn't
    say what happens if there is no presentation modifier. The examples in the table suggest that
    "GMT+01:00" is the full representation, rather than "GMT+1" as in XSLT 2.0; but that's
    only an example. There seems to be no way in which the expected output of this test,
    namely "GMT-05" can be inferred from the normative prose without reference to the examples.
    
    -->
    
    

  <xsl:param name="t" as="xs:dateTime" select="xs:dateTime('0985-03-01T09:15:06.456Z')"/>  

<xsl:template name="main">
<out>;
<xsl:for-each select="-28 to +28">
<x><xsl:value-of select="format-dateTime(adjust-dateTime-to-timezone(
                            $t, .*xs:dayTimeDuration('PT30M')), '[h01][m01][z,2-2]')"/></x>;
</xsl:for-each>                            
</out>
</xsl:template> 


</xsl:stylesheet>