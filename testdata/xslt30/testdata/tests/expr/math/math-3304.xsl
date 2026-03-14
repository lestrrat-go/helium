<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">
<?spec fo#func-round-half-to-even?>
<!-- PURPOSE:  Test of round-half-to-even on floats. -->

<xsl:template match="doc">
<out>
<a b="{round-half-to-even(xs:float(120.0e0))}" 
    c="{round-half-to-even(xs:float(0.0e0))}" 
    d="{round-half-to-even(xs:float(-3.0e0))}"/>
<a b="{round-half-to-even(xs:float(120.0e0), 2)}" 
    c="{round-half-to-even(xs:float(0.0e0), 2)}" 
    d="{round-half-to-even(xs:float(-3.0e0), 2)}"/>
<a b="{round-half-to-even(xs:float(120.0e0), -2)}" 
    c="{round-half-to-even(xs:float(0.0e0), -2)}" 
    d="{round-half-to-even(xs:float(-3.0e0), -2)}"/>
<a b="{round-half-to-even(xs:float(180.0e0), -2)}" 
    c="{round-half-to-even(xs:float(150.0e0), -2)}" 
    d="{round-half-to-even(xs:float(250.0e0), -2)}"/>
<a b="{round-half-to-even(xs:float(-120.0e0), -2)}" 
    c="{round-half-to-even(xs:float(-150.0e0), -2)}" 
    d="{round-half-to-even(xs:float(-250.0e0), -2)}"/>
<a b="{round-half-to-even(xs:float(180.0180e0), 2)}" 
    c="{round-half-to-even(xs:float(150.0150e0), 2)}" 
    d="{round-half-to-even(xs:float(250.0250e0), 2)}"/>
<a b="{round-half-to-even(xs:float(-120.0120e0), 2)}" 
    c="{round-half-to-even(xs:float(-150.0150e0), 2)}" 
    d="{round-half-to-even(xs:float(-250.0250e0), 2)}"/>
</out>
</xsl:template>
</xsl:stylesheet>