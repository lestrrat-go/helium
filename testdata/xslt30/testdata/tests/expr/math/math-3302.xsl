<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec fo#func-round-half-to-even?>
<!-- PURPOSE:  Test of round-half-to-even on decimals. -->

<xsl:template match="doc">
<out>
<a b="{round-half-to-even(120.0)}" c="{round-half-to-even(0.0)}" d="{round-half-to-even(-3.0)}"/>
<a b="{round-half-to-even(120.0, 2)}" c="{round-half-to-even(0.0, 2)}" d="{round-half-to-even(-3.0, 2)}"/>
<a b="{round-half-to-even(120.0, -2)}" c="{round-half-to-even(0.0, -2)}" d="{round-half-to-even(-3.0, -2)}"/>
<a b="{round-half-to-even(180.0, -2)}" c="{round-half-to-even(150.0, -2)}" d="{round-half-to-even(250.0, -2)}"/>
<a b="{round-half-to-even(-120.0, -2)}" c="{round-half-to-even(-150.0, -2)}" d="{round-half-to-even(-250.0, -2)}"/>
<a b="{round-half-to-even(180.0180, 2)}" c="{round-half-to-even(150.0150, 2)}" d="{round-half-to-even(250.0250, 2)}"/>
<a b="{round-half-to-even(-120.0120, 2)}" c="{round-half-to-even(-150.0150, 2)}" d="{round-half-to-even(-250.0250, 2)}"/>
</out>
</xsl:template>
</xsl:stylesheet>